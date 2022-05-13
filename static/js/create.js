async function createNewApplication() {
    //get the url, lang and port
    const url = document.getElementById('giturl').value;
    var select = document.getElementById("lang");
    var lang = select.options[select.selectedIndex].value
    const port = document.getElementById('port').value;

    //check if they are not empty
    if (url == '' || port == '') {
        alert('Please fill in all the fields');
        return
    }

    //do post request to /api/app/new
    const res = await fetch('/api/app/new', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            "github-repo": url,
            "language": lang,
            "port": port
        })
    });
    const data = await res.json();
    if (data.error) {
        if (apps.code == 498) {
            newTokenPair(createNewApplication);
        }
        alert(data.error);
        return
    }
    document.getElementById("result").innerText = "applicazione creata con successo, la locazione é: " + data.data.external_port;
}

async function createNewDatabase() {
    //get the url, lang and port
    const url = document.getElementById('baseDB').value;
    var select = document.getElementById("dbms");
    var dbms = select.options[select.selectedIndex].value

    //do post request to /api/app/new
    const res = await fetch('/api/db/new', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            "databaseName": url,
            "databaseType": dbms
        })
    });
    const data = await res.json();
    if (data.error) {
        if (apps.code == 498) {
            newTokenPair(createNewDatabase);
        }
        alert(data.error);
        return
    }
    document.getElementById("result").innerHTML = "database creato con successo, alcune informazioni: <br>"
        + "importante: la credenziali forntite sono per l'account root<br>"
        + "la password é: " + data.data.pass + "<br>"
        + "la porta é: " + data.data.port + "<br>"
        + "l'utente é: " + data.data.user + "<br>";
}