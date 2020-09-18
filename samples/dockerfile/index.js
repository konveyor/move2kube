import express from 'express'

const port = 8080;

const server = express();
server.use(express.static("public"))
server.listen(port, () => console.log("Listening on port", port));
