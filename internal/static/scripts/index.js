window.onload = function(){
    document.querySelector(".expression-btn").addEventListener("click", sendExpression)
    document.querySelector("#exit").addEventListener("click", Exit)
    document.querySelector(".operations-btn").addEventListener("click", saveDelays)
    showInfo()
    showTasks()
    showWorkers()
    showDelays()
    if (window["WebSocket"]){
        conn = new WebSocket("ws://" + document.location.host + "/ws")
        conn.onmessage = function(evt) {
            const eventData = JSON.parse(evt.data)
            WebSocketUpdates(eventData)
        }
    } else{
        console.log("WebSocket недоступен")
    }
    for (const i of document.querySelectorAll(".menu-btn")){
        i.addEventListener("click", ChangeWindow)
        i.wind = document.querySelector("."+i.dataset.show)
    }
}
const workersList = document.querySelector(".workers-list")
const modal = document.querySelector(".modal")
let windows = document.querySelectorAll(".window")

window.onclick = function(event) {
    if (event.target == modal) {
      modal.style.display = "none";
    }
}

async function Exit(){
    try {
        const response = await fetch(window.location.origin + "/logout", {
            method: "GET",
        });
        if (response.status==200){
            window.location.href = window.location.origin + "/auth"
        }
    } catch (error) {
        throw error
    }
}

function WebSocketUpdates(update){
    if (update["type"] == "update task"){
        UpdateTask(JSON.parse(update["message"]))
    } else if (update["type"] == "update worker"){
        UpdateWorker(JSON.parse(update["message"]))
    }
}

function UpdateWorker(info){
    const el = workersList.querySelector('[data-id="' + info["id"] + '"]')
    const infoEl = el.querySelector(".worker-info")
    if (info["state"] == "at work"){
        infoEl.innerHTML = `<p>Статус: at work</p>
                            <p>Подзадача: `+info['exp']+`</p>
                            <p>ID задачи: `+info['taskId']+`</p>`
    } else{
        infoEl.innerHTML = `<p>Статус: in waiting</p>`
    }
}

function UpdateTask(task){
    const el = document.querySelector('[data-id="' + task["taskId"] + '"]')
    const ping = el.querySelector(".lastPing")
    ping.innerText = "Последнее обновление: " + task["lastPing"]
    if (task["isDone"] === 1){
        el.classList.remove("processing")
        el.classList.add("completed")
        const exp = el.querySelector(".expression-value")
        exp.innerText += "="+task["result"]
    }
}

async function getWorkers() {
    try {
        const response = await fetch(window.location.origin + "/getWorkersInfo", {
            method: "GET",
        });
        if (!response.ok) {
            const text = await response.text()
            throw new Error(text)
        }
        return await response.json()
    } catch (error) {
        throw error
    }
}

async function showDelays() {
    try {
        const data = await getDelays();
        for (let i in data){
            document.getElementById(i).value = data[i]
        }
    } catch (error) {
        const btn = document.querySelector("#settings-btn")
        btn.disabled = true
    }
}

async function getDelays() {
    try {
        const response = await fetch(window.location.origin + "/getDelays", {
            method: "GET",
        });
        if (!response.ok) {
            const text = await response.text()
            console.log(response)
            throw new Error(text)
        }
        return await response.json()
    } catch (error) {
        throw error
    }
}

function saveDelays(){
    const operations = {}
    for (const i of document.querySelectorAll(".operation")){
        operations[i.id] = +i.value
    }
    fetch(window.location.origin + "/updateDelays",{
        body: JSON.stringify({"delays": operations}),
        method: "POST",
        headers:{
            "Content-Type": "application/json"
        }
    }).then(response => {
        if (!response.ok) {
            return response.text().then(text => Promise.reject(text));
        }
        return "";
    })
    .then(data => {
        showNotification("Обновлено");
        delays = data;
    })
    .catch(error => {
        showNotification(error);
    });
}

async function showWorkers() {
    try {
        const data = await getWorkers();
        for (const i of data) {
            workersList.prepend(CreateWorker(i))
        }
    } catch (error) {
        const btn = document.querySelector("#workers-btn")
        btn.disabled = true
    }
}

async function showTasks() {
    try {
        const expressionsList = document.querySelector(".expressions-list")
        const data = await getTasks();
        for (const expression of data) {
            expressionsList.prepend(CreateExpression(expression))
        }
    } catch (error) {
       showNotification(error)
    }
}

async function showInfo(){
    try {
        const name = document.getElementById("profile-name")
        const data = await getInfo();
        name.innerText = data
    } catch (error) {
        showNotification(error)
    }
}

async function getTasks() {
    try {
        const response = await fetch(window.location.origin + "/getTasks", {
            method: "GET",
        });
        if (!response.ok) {
            if (response.status === 401) {
                window.location.href = window.location.origin + "/auth"
                return
            }
            const text = await response.text()
            throw new Error(text)
        }
        return await response.json()
    } catch (error) {
        throw error
    }
}

async function getInfo() {
    try {
        const response = await fetch(window.location.origin + "/getInfo", {
            method: "GET",
        });
        if (!response.ok) {
            if (response.status === 401) {
                window.location.href = window.location.origin + "/auth"
                return
            }
            const text = await response.text()
            throw new Error(text)
        }
        return await response.text()
    } catch (error) {
        throw error
    }
}

function showNotification(text) {
    const notification = document.getElementById("notification");
    notification.innerText = text
    notification.className = "notification show";
    setTimeout(function() {
      notification.className = notification.className.replace("show", "");
    }, 3000);
  }

function sendExpression(){
    const expression = document.querySelector(".expression-value").value
    if (!expression.trim()){
        showNotification("Выражение пустое")
        return
    }
    fetch(window.location.origin + "/addTask",{
        body: JSON.stringify({"task": expression}),
        method: "POST",
        headers:{
            "Content-Type": "application/json"
        }
    }).then(response => {
        if (!response.ok) {
            return response.text().then(text => Promise.reject(text));
        }
        return response.json();
    })
    .then(data => {
        showNotification("Выражение принято");
        const expressionsList = document.querySelector(".expressions-list")
        data['lastPing'] = ""
        expressionsList.prepend(CreateExpression(data));
    })
    .catch(error => {
        showNotification(error);
    });
}

function CreateExpression(expression) {
    const result = document.createElement("li") 
    const div = document.createElement("div")
    value = document.createElement("p")
    value.classList.add("expression-value")
    value.innerText = expression["expression"]
    if (expression["status"] == "completed"){
        value.innerText += "=" + expression["result"]
    }
    div.dataset.id = expression["id"]
    div.append(value)
    const info = document.createElement("div")
    info.insertAdjacentHTML("beforeend", `<p class="id">ID: ${expression["id"]}</p>`)
    info.insertAdjacentHTML("beforeend", `<p class="lastPing">Последнее обновление: ${expression["lastPing"]}</p>`)
    const moreInfo = document.createElement("p")
    moreInfo.classList.add("moreinfo")
    moreInfo.dataset.id = expression["id"]
    moreInfo.innerText = "Больше информации"
    //info.insertAdjacentHTML("beforeend", `<p class="moreinfo" data-id="${expression["id"]}">Больше информации</p>`)
    moreInfo.addEventListener("click", getExpressionInfo)
    info.append(moreInfo)
    info.classList.add("info")
    div.append(info)
    div.classList.add(expression["status"])
    div.classList.add("item")
    result.append(div)
    return result
}

function CreateWorker(worker) {
    const result = document.createElement("li")
    result.classList.add("workers-list-item")
    const div = document.createElement("div")
    div.dataset.id = worker["id"]
    div.innerHTML = `<p class="worker-name">Worker_${worker["id"]}</p>`
    const workerinfo = document.createElement("div")
    workerinfo.classList.add("worker-info")
    workerStatus = document.createElement("p")
    workerStatus.innerHTML = `Статус: ${worker["status"]}`
    workerinfo.append(workerStatus)
    if (worker["expression"] != ""){
        workerExpression = document.createElement("p")
        workerExpression.innerText = worker["expression"]
        workerExpression.innerHTML = `Подзадача: ${worker["expression"]}`
        workerExpressionId = document.createElement("p")
        workerExpressionId.innerHTML = `ID задачи: ${worker["expressionId"]}`
        workerinfo.append(workerExpression)
        workerinfo.append(workerExpressionId)
    }
    div.append(workerinfo)
    result.append(div)
    return result
}

function getExpressionInfo(id){
    modal.style.display = "block"
    getTask(+this.dataset.id).then(data => showExpressionInfo(data)).catch(error => showNotification(error))
}

function getTask(id){
    const data = fetch(window.location.origin + "/getTask"+"?id="+id, {
        method: "GET",
    }).then(response => {if (!response.ok) {
        return response.text().then(text => Promise.reject(text));
    }
    return response.json();})
    return data
}

function showExpressionInfo(task){
    const content = document.querySelector(".modal-content")
    const list = document.querySelector(".subtasks-list")
    list.innerHTML = ""
    if (task["subtasks"]){
        for (const i of task["subtasks"]){
            const li = document.createElement("li")
            li.innerHTML = `<pre>${i["value"]} &rarr; ${i["result"]}            ${i["time"]}</pre>`
            list.append(li)
        }
    }
    content.querySelector(".expression-id").innerText = "ID: " + task["id"]
    content.querySelector(".expression-status").innerText = "Статус: " + task["status"]
    content.querySelector(".expression-lastPing").innerText = "Последнее обновление: " + task["lastPing"]
    content.querySelector(".expression-created").innerText = "Создан: " + task["created"]
    content.querySelector(".expression-result").innerText = "Результат: " + task["result"]
    content.querySelector(".expression-info-value").innerText = task["expression"]
}

function ChangeWindow(){
    for (const i of windows){
        if (!i.classList.contains("hide")){
            i.classList.add("hide")
        }
    }
    this.wind.classList.remove("hide")
}