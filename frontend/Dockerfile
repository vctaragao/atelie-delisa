# Imagem de produção: compila os arquivos estáticos e os serve com nginx,
# que também faz proxy de /api para o backend.
FROM node:22-alpine AS build

WORKDIR /app
COPY package.json ./
RUN npm install

COPY . .
RUN npm run build

FROM nginx:1.27-alpine

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
