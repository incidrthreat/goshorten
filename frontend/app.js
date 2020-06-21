import Navigo from "navigo"

const router = new Navigo()

router
    .on("/", function() {
        document.body.innerHTML = "Home"
    })
    .resolve()