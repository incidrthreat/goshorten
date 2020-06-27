import { shortClient } from '../app'
import { ShortURLReq } from '../pb/url_service_grpc_web_pb'

function Home() {
    document.body.innerHTML = ""
    const homeDiv = document.createElement('div')
    homeDiv.classList.add('home-div')

    const urlLabel = document.createElement('h1')
    urlLabel.innerText = "URL Shortener"
    homeDiv.appendChild(urlLabel)

    const getcodeForm = document.createElement('form')

    const longurlInput = document.createElement('input')
    longurlInput.setAttribute('type', 'text')
    longurlInput.setAttribute('placeholder', 'https://www.alongurl.com/that/should/be/short')
    getcodeForm.appendChild(longurlInput)

    const submitBtn = document.createElement("button")
    submitBtn.innerText = "Shorten dat bish!"
    getcodeForm.appendChild(submitBtn)

    getcodeForm.addEventListener('submit', event => {
        var pattern = /(http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?/

        if(!pattern.test(longurlInput.value)){
            alert(longurlInput.value + " is not a valid URL.")
            return Home()
        }

        event.preventDefault()
        let req = new ShortURLReq()
        req.setLongurl(longurlInput.value)
        shortClient.createURL(req, {}, (err, res) => {
            if(err) {
                alert(err.message)
                return Home()
            }
            alert(window.location.href + res.getShorturl())
            return Home()
        })
    })
    
    homeDiv.appendChild(getcodeForm)

    document.body.appendChild(homeDiv)
}

export { Home }