const express = require('express');
const ws = require('ws');

const app = express();

const wsServer = new ws.Server({ noServer: true});

console.log("Hello World");

wsServer.on('connection', socket => {
	socket.on('message', message =>{
		console.log(message);
	})
});


const server = app.listen('1234')
server.on('upgrade', (request, socket, head) =>{
	wsServer.handleUpgrade(request, socket, head, socket =>{
		wsServer.emit('connection', socket, request);
	});
});