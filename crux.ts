// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

interface TabFunc {
	(name: string, link: HTMLLinkElement);
}

function forEachTab(fn: TabFunc) {
	var tablinks = document.getElementsByClassName("navtab");
	for(var i = 0; i < tablinks.length; i++) {
		var tablink = <HTMLLinkElement>tablinks.item(i);
		var name = <string>tablink.innerHTML;
		fn(name, tablink);
	}
}

function tabBody(name: string): HTMLDivElement {
	return <HTMLDivElement>document.getElementById("nav" + name);
}

function navgo(name: string) {
	forEachTab(function(tname: string, link: HTMLLinkElement) {
		var body = tabBody(tname);
		if (tname == name) {
			body.style.display = "block";
			link.classList.add("navtabcurrent");
		} else {
			body.style.display = "none";
			link.classList.remove("navtabcurrent");
		}
	});
	if (name == "stats") {
		loadLogNames();
	}
}

function handleLog(text: string) {
	var main = document.getElementById("main");
	var atBottom = main.scrollHeight - main.scrollTop == main.clientHeight;
	main.innerHTML = "<pre id=\"log\">" + text + "</pre>";
	if (atBottom) {
		main.scrollTop = main.scrollHeight - main.clientHeight;
	}
}

function loadLog(logname: string) {
	console.log("loadLog", logname);
	var xhr = new XMLHttpRequest();
	// TODO: offset, limit
	xhr.open("GET", "/debug/crux/logs/" + logname);
	xhr.onload = (ev) => {
		if (xhr.status < 200 || xhr.status >= 300) {
			console.log("bad data:", xhr);
			return;
		}
		handleLog(xhr.responseText);
	}
	xhr.onerror = (ev) => { xhrError("log", ev); };
	xhr.send();
}

var curRefreshID;

function handleLogNames(files: Array<string>) {
	var logslist = <HTMLUListElement>document.getElementById("logslist");
	logslist.innerHTML = "";
	for (var file of files) {
		logslist.innerHTML += "<li><a class=\"logslink\">" + file + "</a></li>";
	}
	for (var i = 0; i < logslist.children.length; i++) {
		var li = <HTMLLIElement>logslist.children[i];
		li.children[0].addEventListener('click', function(e: Event) {
			var name = (<HTMLLIElement>e.target).innerHTML;
			loadLog(name);
			window.clearInterval(curRefreshID);
			curRefreshID = window.setInterval(loadLog, 1000, name);
		});
	}
}

function loadStats() {
	var xhr = new XMLHttpRequest();
	xhr.open("GET", "/debug/crux/stats");
	xhr.onload = (ev) => {
		if (xhr.status < 200 || xhr.status >= 300) {
			console.log("bad data:", xhr);
			return;
		}
		var main = document.getElementById("main");
		main.innerHTML = xhr.responseText;
	}
	xhr.onerror = (ev) => { xhrError("stats", ev); };
	xhr.send();

	window.clearInterval(curRefreshID);
	//curRefreshID = window.setInterval(loadStats, 5000);
}

function xhrError(name: string, ev: Event) {
	// TODO: visible user feedback
	console.log("loading "+name+" failed: ", ev);
}

function loadLogNames() {
	var xhr = new XMLHttpRequest();
	xhr.open("GET", "/debug/crux/logs/list");
	xhr.onload = (ev) => {
		if (xhr.status < 200 || xhr.status >= 300) {
			console.log("bad data:", xhr);
			return;
		}
		var res = JSON.parse(xhr.responseText);
		handleLogNames(res.Files);
	}
	xhr.onerror = (ev) => { xhrError("logs list", ev); };
	xhr.send();
}

window.onload = function() {
	forEachTab(function(name: string, link: HTMLLinkElement) {
		link.addEventListener('click', function (e: Event) {
			navgo(name);
		});
	});
	var showstats = document.getElementById("showstats");
	showstats.addEventListener("click", function(e: Event) {
		loadStats();
	});

	navgo("stats");
	loadStats();
};
