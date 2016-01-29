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

interface PageFunc {
	(name: string, page: HTMLDivElement);
}

function forEachPage(fn: PageFunc) {
	var main = document.getElementById("main");
	for(var i = 0; i < main.children.length; i++) {
		var page = <HTMLDivElement>main.children[i];
		fn(page.id, page);
	}
}

function togglePage(pagename: string) {
	forEachPage(function(name: string, page: HTMLDivElement) {
		if (name == pagename) {
			if (page.style.display != "block") {
				page.style.display = "block";
			}
		} else {
			page.style.display = "none";
		}
	});

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
	} else if (name == "goroutines") {
		loadGoroutines();
	}
}

function xhrSend(path: string, fn: EventListener) {
	if (xhr != null) {
		xhr.abort();
	}
	var xhrThis = new XMLHttpRequest();
	xhr = xhrThis;
	xhr.open("GET", path);
	xhr.onload = (ev) => {
		if (xhrThis != xhr) {
			return;
		}
		if (xhr.status < 200 || xhr.status >= 300) {
			console.log("bad data from "+ path+ ":", xhr);
			return;
		}
		fn(ev);
		xhr = null;
	}
	xhr.onerror = (ev) => { xhrError(path, ev); };
	xhr.send();

	window.clearInterval(curRefreshID);
}

function loadLog(logname: string) {
	console.log("loadLog", logname);
	togglePage("mainlogs");

	// TODO: offset, limit
	xhrSend("/debug/crux/logs/" + logname, (ev) => {
		var main = document.getElementById("main");
		var atBottom = main.scrollHeight - main.scrollTop == main.clientHeight;
		var mainlogs = document.getElementById("mainlogs");
		mainlogs.innerHTML = "<pre id=\"log\">" + xhr.responseText + "</pre>";
		if (atBottom) {
			main.scrollTop = main.scrollHeight - main.clientHeight;
		}
	});
	curRefreshID = window.setInterval(loadLog, 1000, logname);
}

var curRefreshID;
var xhr : XMLHttpRequest = null;

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
		});
	}
}

function loadStats() {
	togglePage("mainstats");

	xhrSend("/debug/crux/stats", (ev) => {
		document.getElementById("mainstats").innerHTML = xhr.responseText;
	});
	curRefreshID = window.setInterval(loadStats, 2000);
}

function xhrError(name: string, ev: Event) {
	// TODO: visible user feedback
	console.log("loading "+name+" failed: ", ev);
}

// TODO: merge this into the stats handler.
function loadLogNames() {
	xhrSend("/debug/crux/logs/list", (ev) => {
		var res = JSON.parse(xhr.responseText);
		handleLogNames(res.Files);
	});
}

function loadGoroutines() {
	xhrSend("/debug/crux/goroutines", (ev) => {
		document.getElementById("navgoroutines").innerHTML = xhr.responseText;
	});
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
