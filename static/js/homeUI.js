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
    document.cookie = "accessToken=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    document.cookie = "refreshToken=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    window.location.href = "/login";
};

function logoutconfirm(x) {
    if (x) {
        document.querySelector("#logoutconfirm").style.display = "flex";
        console.log("vedi");
    } else {
        document.querySelector("#logoutconfirm").style.display = "none";
        console.log("nasc");
    }
};

/*######## add document #########*/
let lastPressed;

$("#inputFile").css('display', 'none');

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

function adddoc(type) {
    lastPressed = type;
    $("#inputFile").trigger('click');
}