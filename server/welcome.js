var figlet = require('figlet');

function display() {
	figlet.text('Remote Logger', {
	    font: 'ANSI Shadow',
	    horizontalLayout: 'default',
	    verticalLayout: 'default',
	    width: 80,
	    whitespaceBreak: true
	}, function(err, data) {
	    if (err) {
	        console.log('Something went wrong...');
	        console.dir(err);
	        return;
	    }
	    console.log(data);
	});
}

module.exports = { display };