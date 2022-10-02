// loadDatabases();
// loadApplications();

async function loadDatabases() {
    const res = await fetch('/api/user/getApps/database');
    const dbs = await res.json();
    if (dbs.error) {
        if (dbs.code === 498) {
            await newTokenPair(loadDatabases);
        }
        alert(dbs.error);
        return
    }
    //generate the databases
    if (dbs.data === null) {
        document.getElementById('databasesContainer').innerHTML += '<center><h3>You have no databases</h3></center>';
        return
    }

    document.getElementById('databasesContainer').innerHTML = "";
    for (let i = 0; i < dbs.data.length; i++) {
        const db = dbs.data[i];

        const dbDiv = document.createElement('div');
        dbDiv.id = db.containerID;
        dbDiv.className = 'doc';

        const name = document.createElement('p');
        name.innerText = db.name;

        const exportBtn = document.createElement('button');
        exportBtn.type = "button";
        exportBtn.className = "btn btn-info";
        exportBtn.innerText = "Export";
        exportBtn.disabled = true
        // exportBtn.onclick = exportDB(db.containerID);

        const deleteBtn = document.createElement('button');
        deleteBtn.type = "button";
        deleteBtn.className = "btn btn-danger";
        deleteBtn.innerText = "Delete";
        deleteBtn.setAttribute("onclick", "deleteContainer('" + db.containerID + "')")


        dbDiv.appendChild(name);
        dbDiv.appendChild(exportBtn);
        dbDiv.appendChild(deleteBtn);

        document.getElementById('databasesContainer').appendChild(dbDiv);
    }
}

async function deleteContainer(containerId) {
    const res = await fetch('/api/container/delete/' + containerId, {
        method: 'DELETE'
    });
    const data = await res.json();
    if (data.error) {
        if (data.code === 498) {
            await newTokenPair(deleteContainer, containerId);
        }
        alert(data.error);
        return
    }
    //remove the container from the DOM
    document.getElementById(containerId).remove();
}

async function loadApplications() {
    const res = await fetch('/api/user/getApps/web');
    const apps = await res.json();
    if (apps.error) {
        if (apps.code === 498) {
            await newTokenPair(loadApplications);
        }
        alert(apps.error);
        return
    }

    if (apps.data === null) {
        document.getElementById('applicationsContainer').innerHTML = '<center><h3>You have no applications</h3></center>';
        return
    }
    document.getElementById('applicationsContainer').innerHTML = "";
    for (let i = 0; i < apps.data.length; i++) {
        const app = apps.data[i];

        const appDiv = document.createElement('div');
        appDiv.id = app.containerID;
        appDiv.className = 'doc';

        const name = document.createElement('p');
        name.innerHTML = `<a target="_blank" href="http://vano.my-wan:${app.externalPort}">${app.name}</a>`;

        const publicBtn = document.createElement('button');
        publicBtn.id = "public" + app.containerID;
        publicBtn.type = "button";
        if (app.isPublic) {
            publicBtn.className = "btn btn-warning";
            publicBtn.innerText = "Make private";
            publicBtn.setAttribute("onclick", "makePrivate('" + app.containerID + "')");
        } else {
            publicBtn.className = "btn btn-info";
            publicBtn.innerText = "Make public";
            publicBtn.setAttribute("onclick", "makePublic('" + app.containerID + "')")
        }

        const deleteBtn = document.createElement('button');
        deleteBtn.type = "button";
        deleteBtn.className = "btn btn-danger";
        deleteBtn.innerText = "Delete";
        deleteBtn.setAttribute("onclick", "deleteContainer('" + app.containerID + "')")

        const hr = document.createElement('hr');

        const desc = document.createElement('h5');
        desc.innerText = app.description;

        appDiv.appendChild(name);
        appDiv.appendChild(publicBtn);
        appDiv.appendChild(deleteBtn);
        appDiv.appendChild(hr);
        appDiv.appendChild(desc);

        document.getElementById('applicationsContainer').appendChild(appDiv);
    }
}

async function makePublic(containerId) {
    const res = await fetch('/api/container/publish/' + containerId, {
    });
    const data = await res.json();
    if (data.error) {
        if (data.code === 498) {
            await newTokenPair(makePublic, containerId);
        }
        alert(data.error);
        return
    }
    document.getElementById("public" + containerId).className = "btn btn-warning";
    document.getElementById("public" + containerId).innerText = "Make private";
    document.getElementById("public" + containerId).setAttribute("onclick", "makePrivate('" + containerId + "')");
}

async function makePrivate(containerId) {
    const res = await fetch('/api/container/revoke/' + containerId, {
    });
    const data = await res.json();
    if (data.error) {
        if (data.code === 498) {
            await newTokenPair(makePublic, containerId);
        }
        alert(data.error);
        return
    }
    document.getElementById("public" + containerId).className = "btn btn-info";
    document.getElementById("public" + containerId).innerText = "Make public";
    document.getElementById("public" + containerId).setAttribute("onclick", "makePublic('" + containerId + "')");
}

function create(what) {
    if (what === "db") {
        window.location.href = "/user/database/new";
    } else {
        window.location.href = "/user/application/new";
    }
}