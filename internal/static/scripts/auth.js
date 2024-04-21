window.onload = function(){
    regBlock = document.querySelector(".register-block")
    loginBlock = document.querySelector(".login-block")
    const loginBtn = document.getElementById("login-btn")
    loginBtn.addEventListener("click", Login)
    const regBtn = document.getElementById("register-btn")
    regBtn.addEventListener("click", Register)
}

function Change(){
    loginBlock.classList.toggle("hide")
    regBlock.classList.toggle("hide")
}

function Register(){
    const name = document.getElementById("register-name-input").value
    const password = document.getElementById("register-password-input").value
    const password2 = document.getElementById("register-password2-input").value
    RegisterRequest(name.trim(), password.trim(), password2.trim()).then(data => console.log(data)).catch(error => showNotification(error))
}

function RegisterRequest(name, password, password2){
    const data = fetch(window.location.origin + "/register", {
        method: "POST",
        body: JSON.stringify({"name": name, "password": password, "password2": password2}),
        headers: {
            "Content-Type": "application/json"
        }
    }).then(response => {if (!response.ok) {
        return response.text().then(text => Promise.reject(text));
    }
    return response.text();})
    return data
}


function Login(){
    const name = document.getElementById("login-name-input").value
    const password = document.getElementById("login-password-input").value
    LoginRequest(name.trim(), password.trim()).then(data => {console.log(data); window.location.href = window.location.origin}).catch(error => showNotification(error))
    
}

function LoginRequest(name, password){
    const data = fetch(window.location.origin + "/login", {
        method: "POST",
        body: JSON.stringify({"name": name, "password": password}),
        headers: {
            "Content-Type": "application/json"
        }
    }).then(response => {if (!response.ok) {
        return response.text().then(text => Promise.reject(text));
    }
    return response.text();})
    return data
}

function showNotification(text) {
    const notification = document.getElementById("notification");
    notification.innerText = text
    notification.className = "notification show";
    setTimeout(function() {
      notification.className = notification.className.replace("show", "");
    }, 3000);
  }