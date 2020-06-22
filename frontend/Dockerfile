FROM node:14.4.0-alpine

RUN npm i -g parcel-bundler

RUN mkdir /app

ENV PARCEL_WORKERS=1

WORKDIR /app/

COPY . /app/

RUN npm install && npm fund

EXPOSE 1234 3000

CMD ["parcel", "./index.html", "--hmr-port", "3000"]