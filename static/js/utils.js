async function newTokenPair(callback, ...datas) {
    //get request to get the token pair
    const res = await fetch('/api/tokens/new');
    const data = await res.json();
    if (data.error) {
        alert(data.error);
        return
    }
    callback(...datas);
}

function debounce(func, timeout = 600){
    let timer;
    return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => { func.apply(this, args); }, timeout);
    };
}