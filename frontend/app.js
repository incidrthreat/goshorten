import Navigo from "navigo"
import { ShortenerClient } from "./pb/url_service_grpc_web_pb"

console.log(ShortenerClient)

const router = new Navigo()

router
    .on("/", function() {
        document.body.innerHTML = "Home"
    })
    .resolve()