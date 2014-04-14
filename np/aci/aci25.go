package aci

import (
	"errors"
	"github.com/pzduniak/aiw3-np-server/environment"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"github.com/pzduniak/aiw3-np-server/utils"
	"github.com/pzduniak/logger"
	"net"
	"strconv"
	"strings"
	"time"
)

const NOT_DETECTED = 50001

var NotEnoughParts = errors.New("Not enough parts in the aCI25 message")
var InvalidHeader = errors.New("Invalid header in the aCI25 message")

func parseToken25(parts []string) (int64, int64, error) {
	// There's not enough parts
	// Makes sure that we won't use a too high index
	if len(parts) != 2 {
		return 0, 0, NotEnoughParts
	}

	// Make sure that the prefix is fine
	if parts[0] != "fal" {
		return 0, 0, InvalidHeader
	}

	// Format of the token is:
	// <detection_id>;<guid * detection_id>
	tokenParts := strings.Split(parts[1], ";")

	// Same as above, make sure that there are enough parts
	if len(tokenParts) < 2 {
		return 0, 0, NotEnoughParts
	}

	// Detection ID is a 5-character-long int
	detection_id, err := strconv.ParseInt(tokenParts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	// Then the "GUID" is an int32 * detection_id
	guid, err := strconv.ParseInt(tokenParts[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	// Calculate the actual guid
	guid = guid / detection_id

	// Return data
	return detection_id, guid, nil
}

func SliceIntToString(slice []int) string {
	encoded := ""

	for index, element := range slice {
		encoded += strconv.Itoa(element)

		if index < len(slice)-1 {
			encoded += ","
		}
	}

	return encoded
}

func StringToSliceInt(encoded string) []int {
	slice := make([]int, 0)
	substrings := strings.Split(encoded, ",")

	for _, substring := range substrings {
		element, err := strconv.Atoi(substring)
		if err == nil {
			slice = append(slice, element)
		}
	}

	return slice
}

func IsIntInSlice(slice []int, a int) bool {
	for _, b := range slice {
		if a == b {
			return true
		}
	}

	return false
}

func isHWIDBanned(hwid int64) bool {
	// Query for data on the HWID
	var rows []*struct {
		Banned bool
	}

	err := environment.Env.Database.Query(
		"SELECT banned FROM aci_hwid WHERE hwid = ?",
		hwid,
	).Rows(&rows)

	if err != nil {
		logger.Warning(err)
		return false
	}

	// No data, not detected
	if len(rows) == 0 {
		return false
	}

	// Return whether banned
	return rows[0].Banned
}

func AppendHWID(hwid int64, uid int, detected bool) error {
	// Query for current data on the HWID
	var rows []*struct {
		Uids   string
		Banned bool
	}

	err := environment.Env.Database.Query(
		"SELECT uids, banned FROM aci_hwid WHERE hwid = ?",
		hwid,
	).Rows(&rows)

	if err != nil {
		return err
	}

	// Insert a new row
	if len(rows) == 0 {
		err = environment.Env.Database.Query(
			"INSERT INTO aci_hwid (hwid, uids, banned) VALUES(?, ?, ?)",
			hwid,
			SliceIntToString([]int{uid}),
			detected,
		).Run()

		if err != nil {
			return err
		}

		return nil
	}

	current := StringToSliceInt(rows[0].Uids)

	if !IsIntInSlice(current, uid) {
		current = append(current, uid)

		err = environment.Env.Database.Query(
			"UPDATE aci_hwid SET uids = ?, banned = ? WHERE hwid = ?",
			SliceIntToString(current),
			detected,
			hwid,
		).Run()

		if err != nil {
			return err
		}
	}

	if rows[0].Banned == false && detected {
		err = environment.Env.Database.Query(
			"UPDATE aci_hwid SET banned = ? WHERE hwid = ?",
			true,
			hwid,
		).Run()

		if err != nil {
			return err
		}
	}

	return nil
}

func HandleCI25(
	conn net.Conn,
	connection_data *structs.ConnData,
	packet_data *structs.PacketData,
	stringParts []string,
) error {

	// Parse the token
	detection_id, guid, err := parseToken25(stringParts)
	if err != nil {
		return err
	}

	logger.Infof("DETECTION FROM %s: %d", connection_data.Username, detection_id)

	// We already know that the user is unclean, don't waste time handling it.
	if connection_data.IsUnclean {
		return nil
	}

	// check the hwid for bans
	if isHWIDBanned(guid) {
		detection_id = 10100
	}

	// If reason equals NOT_DETECTED, then it's a check
	if detection_id == NOT_DETECTED || detection_id == 41009 {
		// Client is checked, allow it to connect
		connection_data.LastCI = time.Now()

		// Append the GUID
		err = AppendHWID(guid, structs.NpidToId(connection_data.Npid), false)
		if err != nil {
			return err
		}
	} else {
		// Set the connection to unclean
		connection_data.IsUnclean = true

		// Append the GUID
		err = AppendHWID(guid, structs.NpidToId(connection_data.Npid), true)
		if err != nil {
			return err
		}

		if environment.Env.Config.NP.AnticheatInstant {
			// Ban that user
			err = utils.BanUser(connection_data.Username, detection_id, time.Hour*24*14)
			if err != nil {
				return err
			}

			// Kick him!
			if connection_data.ServerId != 0 {
				err = utils.KickUser(connection_data.ServerId, connection_data.Npid, detection_id)
			}
		} else {
			// Add a delayed ban
			err = utils.AddDelayedBan(connection_data.Npid, detection_id, guid)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
