/*######## SETUP HOMEPAGE #########*/
setuphomepage();

async function setuphomepage() {
    //se ho viaggi in programma
    // // let travel = await gettravel();
    // if (travel !== undefined) {
    //     //se mancano dei documenti
    //     if (docsmissing()) {
    //         document.querySelector("#c_check").style.display = "block";
    //         document.querySelector("#c_newtravel").style.display = "none";
    //     } else {
    //         document.querySelector("#c_travel").style.display = "block";
    //     }
    // } else { //se non ho viaggi in programma
    //     document.querySelector("#c_newtravel").style.display = "block";
    //     document.querySelector("#c_check").style.display = "none";
    // }
}

/*######## UPLOAD DOCUMENTS #########*/

async function uploadFile() {
    // const formData = new FormData();
    // let file = document.getElementById("inputFile").files[0]
    // formData.append("document", file);
    // const response = await fetch("/upload/file", {
    //     method: 'POST',
    //     body: formData
    // });
    // let resp = await response.json();
    // console.log(resp)
    // return {
    //     id: resp.fileID, name: file.name
    // }
}

async function uploadInfo(type) {
    // let fileData = await uploadFile()
    // console.log(fileData)
    // let body = {
    //     user: user,
    //     documentInfo: {
    //         id: fileData.id,
    //         title: fileData.name,
    //         type: type
    //     }
    // }
    // console.log(body)

    // const response = await fetch("/upload/info/" + fileData.id, {
    //     method: 'POST',
    //     body: JSON.stringify(body)
    // });
    // let resp = await response.json();
    // console.log(resp)
}

/*######## TRAVEL #########*/

// async function newtravel() {
//     let travel = document.querySelector("#travelDestination").value; // 
//     const response = await fetch("/travel/update", {
//         method: 'POST',
//         body: JSON.stringify(user)
//     });
//     const resp = await response.json();
//     user.travelTo = travel;
//     console.log(resp);
//     setuphomepage();
// }

// async function gettravel() {
//     const response = await fetch("/travel/get", {
//         method: 'POST',
//         body: JSON.stringify(user)
//     });
//     const resp = await response.json();
//     console.log(resp)
//     return resp.travel
// }

function docsmissing() {
    /*
    let req;
    displays.forEach(element => {
        let travel = await gettravel();
        if(element.name==travel){
            req=element;
        }
    });
    let missing=1;
    
    user.documents
    if(req.id )*/
    return true;
}