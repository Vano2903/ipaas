let lastInsertedUrl = '';
$("#giturl").keyup(debounce(async () => {
    //get the url
    const url = document.getElementById('giturl').value;
    if (url === '') {
        return
    }
    if (url === lastInsertedUrl) {
        return
    }
    lastInsertedUrl = url;

    const res = await fetch('/api/user/validate', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            "repo": url,
        })
    });

    const data = await res.json();
    if (data.error) {
        if (data.code === 498) {
            await newTokenPair(createNewApplication);
        }else {
            $("#branches").addClass("is-invalid");
            $("#branches").removeClass("is-valid");
            $("#branches").val("");
            $("#desc").val("");

            $("#err-github").text(data.msg);
            $("#err-github").show();
            $("#giturl").removeClass("is-valid");
            $("#giturl").addClass("is-invalid");
        }
        return
    }

    $("#err-github").hide();
    $("#giturl").addClass("is-valid");
    $("#giturl").removeClass("is-invalid");

    $("#desc").val(data.data.description);
    $("#branches").removeClass("is-invalid");
    $("#branches").addClass("is-valid");

    const defaultBranch = data.data.defaultBranch;
    const branches = data.data.branches;
    const select = document.getElementById("branches");
    select.innerHTML = '';
    for (let i = 0; i < branches.length; i++) {
        const option = document.createElement("option");
        option.selected = branches[i] === defaultBranch;
        option.value = branches[i];
        option.text = branches[i];
        select.appendChild(option);
    }
}));

function getEnvs() {
    const envsKeys = $(".env-key");
    const envsValues = $(".env-value");
    const envs = [];
    for (let i = 0; i < envsKeys.length; i++) {
        if (envsKeys[i].value === '' || envsValues[i].value === '') {
            continue
        }
        envs.push({
            key: envsKeys[i].value,
            value: envsValues[i].value
        })
    }
    return envs;
}

function thereAreEnvs() {
    const envsKeys = $(".env-key");
    return envsKeys.length > 0;
}

async function createNewApplication() {
    const select = document.getElementById("lang");
    const select2 = document.getElementById("branches");
    //get the url, lang and port
    const url = document.getElementById('giturl').value;
    const branch = select2.options[select2.selectedIndex].value;
    const lang = select.options[select.selectedIndex].value
    const port = document.getElementById('port').value;
    const description = document.getElementById('desc').value;
    appObj = {
        "github-repo": url,
        "github-branch": branch,
        "language": lang,
        "port": port,
        "description": description,
    }

    if (thereAreEnvs() ) {
        appObj.envs = getEnvs();
    }

    //check if they are not empty
    if (url === '' || port === '') {
        alert('Please fill in all the fields');
        return
    }

    $("#creationButton").prop("disabled", true);
    document.getElementById("result").innerText = "Stiamo creando la tua applicazione...";
    //do post request to /api/app/new
    const res = await fetch('/api/app/new', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(appObj)
    });
    const data = await res.json();

    $("#creationButton").prop("disabled", false);
    if (data.error) {
        if (data.code === 498) {
            await newTokenPair(createNewApplication);
        }else{
            alert(data.msg);
        }
        return
    }
    document.getElementById("result").innerText = "applicazione creata con successo, la locazione é: " + data.data.external_port;
}

async function createNewDatabase() {
    //get the url, lang and port
    const url = document.getElementById('baseDB').value;
    const select = document.getElementById("dbms");
    const dbms = select.options[select.selectedIndex].value;

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
        if (data.code === 498) {
            await newTokenPair(createNewDatabase);
        }else {
            alert(data.msg);
        }
        return
    }
    document.getElementById("result").innerHTML = "database creato con successo, alcune informazioni: <br>"
        + "importante: la credenziali forntite sono per l'account root<br>"
        + "la password é: " + data.data.pass + "<br>"
        + "la porta é: " + data.data.port + "<br>"
        + "l'utente é: " + data.data.user + "<br>";
}