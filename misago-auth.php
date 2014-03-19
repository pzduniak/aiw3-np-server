<?php
function generate_response($state = false, $message = 'Success.', $uid = 0, $username = 'Anonymous', $email = 'anonymous@example.com', $sid = 0)
{
    $response = '';
    
    if ($state) {
        $response .= 'ok#';
    } else {
        $response .= 'fail#';
    }
    
    $response .= $message . '#' . $uid . '#' . $username . '#' . $email . '#' . $sid . '#';
    return $response;
}

function get_random_string($valid_chars, $length)
{
    // start with an empty random string
    $random_string = "";

    // count the number of chars in the valid chars string so we know how many choices we have
    $num_valid_chars = strlen($valid_chars);

    // repeat the steps until we've created a string of the right length
    for ($i = 0; $i < $length; $i++)
    {
        // pick a random number from 1 up to the number of valid chars
        $random_pick = mt_rand(1, $num_valid_chars);

        // take the random character out of the string of valid chars
        // subtract 1 from $random_pick because strings are indexed starting at 0, and we started picking at 1
        $random_char = $valid_chars[$random_pick-1];

        // add the randomly-chosen char onto the end of our string so far
        $random_string .= $random_char;
    }

    // return our finished random string
    return $random_string;
}

$data = file_get_contents('php://input');
$data = explode('&&', $data);

$username = trim(htmlspecialchars(str_replace(array("\r\n", "\r", "\0"), array("\n", "\n", ''), $data[0]), ENT_COMPAT, 'UTF-8'));
$password = trim(htmlspecialchars(str_replace(array("\r\n", "\r", "\0"), array("\n", "\n", ''), $data[1]), ENT_COMPAT, 'UTF-8'));

if ($username == '' || $password == '' || !$username || !$password) {
    echo generate_response(false, 'Username and/or password is empty.');
    exit;
}

$db = array(
    'host' => '127.0.0.1',
    'database' => 'misago',
    'user' => 'misago',
    'password' => 'password'
);

$db = mysqli_connect($db['host'], $db['user'], $db['password'], $db['database']);

if (!$db) {
    echo generate_response(false, 'An error occoured while connecting to database.');
    exit;
}

$sql = "SELECT u.id AS id, u.username AS username, u.password AS password, u.rank_id AS rank, u.email AS email, b.reason_user AS reason, b.expires AS expires
        FROM misago_user AS u
        LEFT OUTER JOIN misago_ban AS b
        ON (b.test = 0 AND b.ban = u.username OR b.ban = u.email)
        OR (b.test = 1 AND b.ban = u.username)
        OR b.ban = u.email
        WHERE u.username = '" . $username . "'";

$result = mysqli_query($db, $sql);

if (!$result) {
    echo generate_response(false, 'An error occoured while querying the database.');
    exit;
}

if (mysqli_num_rows($result) == 0) {
    echo generate_response(false, 'You have specified an incorrect username.');
    exit;
}

while ($row = mysqli_fetch_assoc($result)) {
    if ($row['reason']) {
        echo generate_response(false, "Your account is banned. Reason: " . $row['reason']);
        exit;
    }
    
    if (strpos($row['password'], "bcrypt$") === false) {
        echo generate_response(false, 'Your account is not compatible with the client. Please log in onto forums.');
        exit;
    }
    
    $verify = password_verify($password, substr($row['password'], 7));
    
    if ($verify) {
        $token = get_random_string('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789', 20);
        
        if (!$token) {
          echo generate_response(false, 'An error occoured while generating session token.');
          exit;
        }
        
        $redis = new Redis();
        
        if (!$redis->connect('127.0.0.1', 6379, 5)) {
          echo generate_response(false, 'An error occoured while accessing the session service.');
          exit;
        }
        
        if ($redis->exists('session:' . $row['id'] . ':*')) {
          if (!$redis->delete('session:' . $row['id'] . ':*')) {
              echo generate_response(false, 'An error occoured while deleting an existsing session.');
              exit;
          }
        }
        
        $session_id = 'session:' . $row['id'] . ':' . $token;
        
        if (!$redis->set($session_id, '')) {
          echo generate_response(false, 'An error occoured while the accessing session service.');
          exit;
        }
        
        echo generate_response(true, 'Success.', $row['id'], $row['username'], $row['email'], $session_id);
        exit;
    } else {
        echo generate_response(false, 'You have specified an incorrect password.');
        exit;
    }
}
?>
