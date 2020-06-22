import Navigo from 'navigo'

import { Home } from './views/home'
import { GetUrl } from './views/geturl'

import { ShortenerClient, ShortURLReq } from './pb/url_service_grpc_web_pb'

const shortClient = new ShortenerClient('http://localhost:8080', null, null)

const router = new Navigo()


router
    .on("/", Home)
    .on("/:code", GetUrl)
    .resolve()

export { router, shortClient }