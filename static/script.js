document.addEventListener("DOMContentLoaded", function () {
    fetch("/api")
        .then(response => response.json())
        .then(data => {
            document.getElementById("api-message").textContent = data.message;
        });
});
