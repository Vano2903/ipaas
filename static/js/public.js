let id = document.getElementById('id').classList;
let lastRender = 0;

async function loadApplications() {
    const res = await fetch('/api/' + id + '/all');
    const apps = await res.json();
    if (apps.error) {
        alert(apps.error);
        return
    }

    if (apps.data === null) {
        lastRender = 0
        document.getElementById('applicationsContainer').innerHTML = '<center><h3>Questo studente non ha applicazioni pubbliche</h3></center>';
        return
    }

    console.log(apps.data.length);
    console.log(lastRender);

    if (apps.data.length === lastRender) {
        console.log('nothing to render');
        return
    }

    lastRender = apps.data.length;
    document.getElementById('applicationsContainer').innerHTML = "";
    for (let i = 0; i < apps.data.length; i++) {
        const app = apps.data[i];

        const appDiv = document.createElement('div');
        appDiv.id = app.containerID;
        appDiv.className = 'doc';

        const name = document.createElement('p');
        name.innerHTML = `<a target="_blank" href="http://${app.externalPort}">${app.name}</a>`;

        const hr = document.createElement('hr');

        const desc = document.createElement('h5');
        desc.innerText = app.description;

        appDiv.appendChild(name);
        appDiv.appendChild(hr);
        appDiv.appendChild(desc);

        document.getElementById('applicationsContainer').appendChild(appDiv);
    }
}

loadApplications();

setInterval(loadApplications, 5000);

// (function poll() {
//     setInterval(loadApplications, 5000);
// })();