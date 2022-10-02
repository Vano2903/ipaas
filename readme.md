[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]

# IPAAS

## Itis Paleocapa As A Service

Itis Paleocapa as a Service (abbreviated IPaaS) is a webapp dedicated to the students of the [I.T.I.S. Pietro Paleocapa](https://www.itispaleocapa.edu.it) of Bergamo.
Through an access guaranteed by [PaleoID](https://github.com/cristianlivella/paleoid-backend), the application allows users to host their web applications on the school server.

**Description:**

As anticipated, the program allows users to distribute their application on the school network servers, thus providing a useful and concrete tool for all the developers inside the institute.
The main difference compared to other competitors in the sector lies in the simplification of use for students. The only requirement is to have an email from the institution, without requiring a credit card to verify its authenticity.
Furthermore, IPaaS does not limit the number of applications that can be hosted by a single user, does not impose a maximum hour limit for hosted applications, and does not require payments or subscriptions of any kind.

### Used technologies

- [Go](https://go.dev): Go is the main programming language as it is used for the entire back-end.
- html/css/js: The front-end is built with the HTML markup language, styling in CSS and application logic in JavaScript and related frameworks.
- [Docker](https://www.docker.com): Thanks to docker the application can containerize databases and applications created by the end users.

### Requirements

- docker-compose
- docker: make sure you have sudo privileges on the docker group (check this out to know how to do so [docker post-installation on linux](https://docs.docker.com/engine/install/linux-postinstall/)), if you don't wanna do that tho then run `go build .` and run the binary as sudo
- required images (to install them run `docker pull <image name>`:
  - golang:1-alpine3.15
  - mysql:8.0.28-oracle
  - mariadb:10.8.2-rc-focal
  - mongo:5.0.6

### How to use

- Make sure to create a .env environent following the .env.example file
- run `$ docker-compose up --build -d`
- go run .

_**for the sorint reviewr i sent an email with a working .env file to hackersgen@sorint.it**_

### Example

you can use this repo [vano2903/testing](https://github.com/Vano2903/testing/) as a testing webserver

### Latest Version

Currently IPaaS is being developed as a microservice application and it's source code can be found [here](https://github.com/ipaas-org).
For now though this repository has a working version, when the microservice version will be stable enough this repo will be archived.

### Credits:

All staff currently involved in the development of this project can be found from the list below.

- [@Vano2903](https://github.com/Vano2903/) as `Founder`, `Project Manager`, `Team Manager`, `Back-end developer`, `Front-end developer`.
- [@davixlive](https://github.com/davixlive) as `Front-end developer`.

If you want to collaborate on the project, feel free to help us by proposing new issues.
If you feel you can make an even greater contribution, consider joining the project development team by [contacting us](https://mail.google.com/mail/?view=cm&source=mailto&to=davidevanoncini2003@gmail.com)

<!-- or making a voluntary donation through our [Ko-fi page](https://ko-fi.com/Vano2903). -->

[stars-shield]: https://img.shields.io/github/stars/vano2903/ipaas.svg?style=for-the-badge
[stars-url]: https://github.com/vano2903/ipaas/stargazers
[issues-shield]: https://img.shields.io/github/issues/vano2903/ipaas.svg?style=for-the-badge
[issues-url]: https://github.com/vano2903/ipaas/issues
[license-shield]: https://img.shields.io/github/license/vano2903/ipaas.svg?style=for-the-badge
[license-url]: https://github.com/vano2903/ipaas/blob/master/LICENSE.txt
[product-screenshot]: images/screenshot.png
