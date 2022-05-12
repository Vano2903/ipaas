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
    localStorage.removeItem("rememberMe");
    localStorage.removeItem("user")
    location.reload();
};

/*######## add document #########*/
let lastPressed;

$("#inputFile").css('display', 'none');

function genDocuments(array) {
    let high = document.querySelector("#documents > #high")
    high.innerHTML = ""
    array.forEach(doc => {
        let div = document.createElement("div")
        div.setAttribute("id", doc.id)
        div.setAttribute("class", "doc")
        let p = document.createElement("p")
        p.innerHTML = doc.type
        let img = document.createElement("img")
        img.setAttribute("src", doc.url)
        div.appendChild(p)
        div.appendChild(img)
        high.appendChild(div)
    });
}

async function docrefresh() {
    const res = await fetch('/document/get/all', {
        method: "POST",
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(user)
    });

    const documents = await res.json();
    genDocuments(documents)
}

function adddoctoggle() {
    document.querySelector("#adddoc").classList.toggle("open");
    if (document.querySelector("#adddoc").classList.contains("open"))
        document.querySelector("#adddoc>h1").innerHTML = "-";
    else
        document.querySelector("#adddoc>h1").innerHTML = "+";
}

function adddoc(type) {
    lastPressed = type;
    $("#inputFile").trigger('click');
}

$("#inputFile").on('change', async function () {
    await uploadInfo(lastPressed)
    docrefresh()
});