package http

import (
	"code.google.com/p/go.crypto/bcrypt"
	"git.cloudrack.io/aiw3/np-server/config"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/storage"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"git.cloudrack.io/aiw3/np-server/utils"
	"github.com/codegangsta/martini"
	"github.com/dchest/uniuri"
	"github.com/fiam/gounidecode/unidecode"
	"github.com/pzduniak/logger"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HTTPServer struct{}

func New() *HTTPServer {
	return &HTTPServer{}
}

// Username and password string trim cutset
const trim_cutset = "\r\n"

func (a *HTTPServer) Start() {
	// Create a new token
	m := martini.Classic()

	// GET / - version info
	m.Get("/", func() string {
		// config.VERSION is a const containing the version string
		return "aiw3/np-server " + config.VERSION
	})

	// Remote kick function. Secure before using.
	/*m.Get("/rkick/:name", func(params martini.Params) string {
			var rows []*struct {
				Id int
			}

			err := environment.Env.Database.Query(`
	SELECT
		id
	FROM
		misago_user
	WHERE
		username = ?`, params["name"]).Rows(&rows)

			if err != nil {
				return err.Error()
			}

			if len(rows) != 1 {
				return "User not found"
			}

			npid := structs.IdToNpid(rows[0].Id)

			connection := storage.GetClientConnection(npid)

			if connection == nil {
				return "Not connected"
			}

			connection.IsUnclean = true

			if connection.ServerId != 0 {
				err = utils.KickUser(connection.ServerId, npid, 10001)
				if err != nil {
					return "Cannot kick user"
				}

				return "User marked as unclean and kicked"
			} else {
				// ehmehgehrd spam with packets
				for _, server := range storage.Servers {
					if server != nil && server.Valid {
						utils.KickUser(server.Npid, npid, 10001)
					}
				}
			}

			return "User not connected to a server; marked as unclean"
		})*/

	// GET /authenticate - remauth replacement
	m.Post("/authenticate", func(r *http.Request) string {
		// Read all data from the body into a variable
		// It might be insecure and cause high memory usage if anyone exploits it
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Error(err)
			return GenerateResponse(false, "Error while reading from POST body", 0, "Anonymous", "anonymous@example.com", "0")
		}

		// As specified in the client, "username" (email in our usecase) and password are seperated by &&
		parts := strings.Split(string(body), "&&")

		// There MUST be two parts or the program will panic!
		if len(parts) != 2 {
			return GenerateResponse(false, "Invalid request", 0, "Anonymous", "anonymous@example.com", "0")
		}

		// Trim newlines from the email and password
		// In original implementation \0 was also removed, but it seems that golang won't compile a const with \0
		email := strings.Trim(parts[0], trim_cutset)
		password := strings.Trim(parts[1], trim_cutset)

		// Not sure if it's performant, but that's just like the example in Jet documentation
		// This struct is used for query result mapping
		var rows []*struct {
			Id       int
			Username string
			Password string
			Rank     int
			Email    string
			Reason   string
			Expires  time.Time
		}

		// The larger query seems to be faster than two seperate ones
		// Might be because Jet's mapper uses reflection which isn't that fast
		err = environment.Env.Database.Query(`
SELECT
	u.id AS id,
	u.username AS username,
	u.password AS password,
	u.rank_id AS rank,
	u.email AS email,
	b.reason_user AS reason,
	b.expires AS expires
FROM 
	misago_user AS u
LEFT OUTER JOIN
	misago_ban AS b
ON
	(b.test = 0 AND b.ban = u.username OR b.ban = u.email)
OR
	(b.test = 1 AND b.ban = u.username)
OR
	b.ban = u.email	
WHERE
	u.email = ?`, email).Rows(&rows)

		// Probably MySQL went down
		if err != nil {
			logger.Error(err)
			return GenerateResponse(false, "Database error", 0, "Anonymous", "anonymous@example.com", "0")
		}

		// There's no email like that!
		if len(rows) != 1 {
			return GenerateResponse(false, "Wrong email or password", 0, "Anonymous", "anonymous@example.com", "0")
		}

		// Put it here, because we already know if the user is banned.
		// TODO: Escape #s, either here or in the GenerateResponse function
		if rows[0].Reason != "" && rows[0].Expires.After(time.Now()) {
			return GenerateResponse(
				false,
				"Your account is banned. Reason: "+strings.Trim(rows[0].Reason, "\n\r#"),
				0,
				"Anonymous",
				"anonymous@example.com",
				"0",
			)
		}

		// I'm too lazy to add the original Django hash implementation. Some people will have to login onto forums,
		// so Django's auth module can rehash their passwords.
		if rows[0].Password[:7] != "bcrypt$" {
			return GenerateResponse(
				false,
				"Your account is not compatible with the client. Please log in onto forums.",
				0,
				"Anonymous",
				"anonymous@example.com",
				"0",
			)
		}

		// I wonder if the bcrypt password checker is timing-attack proof
		err = bcrypt.CompareHashAndPassword([]byte(rows[0].Password[7:]), []byte(password))
		if err != nil {
			return GenerateResponse(false, "Wrong email or password", 0, "Anonymous", "anonymous@example.com", "0")
		}

		// Unidecode the username
		username := unidecode.Unidecode(rows[0].Username)

		token := uniuri.NewLen(20)
		existing, err := environment.Env.Redis.Keys("session:" + strconv.Itoa(rows[0].Id) + ":*").Result()
		if err != nil {
			logger.Warning(err)
			return GenerateResponse(false, "Error while accessing session service", 0, "Anonymous", "anonymous@example.com", "0")
		}

		if len(existing) > 0 {
			removed, err := environment.Env.Redis.Del(existing...).Result()
			if err != nil {
				logger.Warning(err)
				return GenerateResponse(false, "Error while accessing session service", 0, "Anonymous", "anonymous@example.com", "0")
			}

			logger.Debugf("Removed %d old sessions while authenticating %s", removed, username)
		}

		status, err := environment.Env.Redis.Set("session:"+strconv.Itoa(rows[0].Id)+":"+token, structs.StripPort(r.RemoteAddr)+";"+strconv.Itoa(rows[0].Rank)+";"+username).Result()
		if err != nil {
			logger.Warning(err)
			return GenerateResponse(false, "Error while accessing session service", 0, "Anonymous", "anonymous@example.com", "0")
		}

		if status != "OK" {
			logger.Warning("Authentication of %s failed. Redis returned %s on set.", username, status)
			return GenerateResponse(false, "Error while accessing session service", 0, "Anonymous", "anonymous@example.com", "0")
		}

		// Return a "Success response"
		return GenerateResponse(
			true,
			"Success",
			rows[0].Id,
			strings.Trim(username, "\n\r#"),
			rows[0].Email,
			strconv.Itoa(rows[0].Id)+":"+token,
		)
	})

	// Standard HTTP serving method
	logger.Infof("Serving auth server on %s", environment.Env.Config.HTTP.BindingAddress)
	http.ListenAndServe(environment.Env.Config.HTTP.BindingAddress, m)
}

// Format of the response is:
// (ok|fail)#(<error_msg>|Success)#<user_id>#<username>#<email>#<session_id>
func GenerateResponse(state bool, message string, uid int, username string, email string, sid string) string {
	response := ""

	if state == true {
		response += "ok#"
	} else {
		response += "fail#"
	}

	if message != "" {
		response += message + "#"
	} else {
		response += "Success#"
	}

	response += strconv.Itoa(uid) + "#" + username + "#" + email + "#" + sid

	return response
}
