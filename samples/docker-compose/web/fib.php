<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fibonacci</title>
</head>
<body>
    <h1>Answer</h1>
    <p><?php
        $api_url = "http://api:1234/fib?n=";
        $new_url = getenv("API_URL");
        if ($new_url) $api_url = $new_url . "?n=";

        $n = $_GET["n"];
        if (ctype_digit($n)) {
            $n = intval($n);
            $reply = json_decode(file_get_contents($api_url . $n));
            if(property_exists($reply, 'ans')) {
                echo "The " . $n . "th fibonacci number is " . $reply->ans;
            } else {
                echo "Something went wrong. Please try again.";
                echo "Error:" . $reply->error;
            }
        } else {
            echo "You entered an invalid integer.";
        }
        ?></p>
        <a href="/">Go back</a>
</body>
</html>
