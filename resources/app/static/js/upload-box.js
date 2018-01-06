function uploadFile() {
    document.getElementById("text").innerHTML = "<b>" + document.getElementById("FileName").files[0].name + "</b> uploaded <br><br><i class='fa fa-check-circle-o fa-3x check'></i>"
}

function gray() {
	document.getElementById("fileBox").classList.add('is-dragover');
}

function white() {
	document.getElementById("fileBox").classList.remove('is-dragover');
}
