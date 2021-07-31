const express = require('express');
const ws = require('ws');
var List = require("collections/list");

const app = express();
const events = new List()


const wsServer = new ws.Server({ noServer: true});

console.log("Hello World");


wsServer.on('connection', socket => {
	events.clear();
	console.log("clearing events");
	socket.on('message', message =>{
		console.log(message);
		events.add(message)
	})
});


const server = app.listen('1234')
server.on('upgrade', (request, socket, head) =>{
	wsServer.handleUpgrade(request, socket, head, socket =>{
		wsServer.emit('connection', socket, request);
	});
});

app.listen(3000, () => console.log('Application started on port 3000'));

app.set('view engine', 'pug')
app.get('/', function (req, res) {
  res.render('index', { title: 'Hey', message: 'Hello there!', data: events.toArray() })
})
