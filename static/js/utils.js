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