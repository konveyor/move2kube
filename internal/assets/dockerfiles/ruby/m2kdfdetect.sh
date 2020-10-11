if [ ! -f "$1/Gemfile" ]; then
   exit 1
else
   echo '{"Port": 8080, "APPNAME": "app"}'
fi