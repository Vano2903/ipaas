/*######## BASE #########*/
// screen.orientation.lock('portrait');
//not to mess up while scanning a code

//demo start page
transition("mais", "wallet");

function transition(from, to) {
    document.querySelector("#" + from).style.opacity = 0;
    setTimeout(() => {
        document.querySelector("#" + from).style.display = "none";
        document.querySelector("#" + to).style.opacity = 0;
        document.querySelector("#" + to).style.display = "block";
    }, 600);
    setTimeout(() => {
        document.querySelector("#" + to).style.opacity = 1;
    }, 700);
}

function logout() {
    //remove accessToken and refreshToken from cookie
    document.cookie = "ipaas-access-token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    document.cookie = "ipaas-refresh-token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    window.location.href = "/login";
}

function logoutconfirm(x) {
    if (x) {
        document.querySelector("#logoutconfirm").style.display = "flex";
        console.log("vedi");
    } else {
        document.querySelector("#logoutconfirm").style.display = "none";
        console.log("nasc");
    }
}

/*######## add document #########*/
let lastPressed;

$("#inputFile").css('display', 'none');

function adddoc(type) {
    lastPressed = type;
    $("#inputFile").trigger('click');
}

function adddoctoggle() {
    document.querySelector("#adddoc").classList.toggle("open");
    document.querySelector("#adddoc2").classList.toggle("open");
    if (document.querySelector("#adddoc").classList.contains("open") || document.querySelector("#adddoc2").classList.contains("open")) {
        document.querySelector("#adddoc>h1").innerHTML = "-";
        document.querySelector("#adddoc2>h1").innerHTML = "-";
    } else {
        document.querySelector("#adddoc>h1").innerHTML = "+";
        document.querySelector("#adddoc2>h1").innerHTML = "+";
    }
}

function addEnv() {
    const env = document.createElement("div");
    env.className = "env";
    env.id = "env" + document.querySelectorAll(".env").length;
    //put the two input inline
    env.innerHTML = `
        <div class="row">
            <div class="col-5">
                <input type="text" class="form-control env-key" placeholder="Chiave" required>
            </div>
            <div class="col-5">
                <input type="text" class="form-control env-value" placeholder="Valore" required>
            </div>
            <div class="col-1">
                <button class="btn btn-danger" onclick="removeEnv('${env.id}')">X</button>
            </div>
        </div>
        <br>

<!--        // <input type="text" placeholder="Nome variabile" class="form-control envName">-->
<!--        // <input type="text" placeholder="Valore variabile" class="form-control envValue">-->
<!--        // <button onclick="removeEnv('${env.id}')">X</button>-->
    `;
    document.querySelector("#env-container").appendChild(env);
}

function removeEnv(id) {
    document.querySelector("#" + id).remove();
}