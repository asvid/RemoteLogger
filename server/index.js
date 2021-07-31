const express = require('express');
const ws = require('ws');
var List = require("collections/list");
const open = require('open');
const welcome = require("./welcome")

welcome.display();

const app = express();
const events = new List()

const wsServer = new ws.Server({ noServer: true});

wsServer.on('connection', socket => {
	events.clear();
	console.log("clearing events");
	socket.on('message', message =>{
		events.add(JSON.parse(message))
	})
});

const server = app.listen(process.env.WS_PORT || '1234')

console.log("Remote Logger is starting on port: ", process.env.WS_PORT || '1234');

server.on('upgrade', (request, socket, head) =>{
	wsServer.handleUpgrade(request, socket, head, socket =>{
		wsServer.emit('connection', socket, request);
	});
});

var ip = require("ip");
app.listen(3000, () => console.log('Application started on port 3000 at IP:', ip.address()));
open('http://localhost:3000/');

app.set('view engine', 'pug')
app.get('/', function (req, res) {
  res.render('index', { title: 'Remote Logger', message: 'Remote Logger!', data: events.toArray() })
})
