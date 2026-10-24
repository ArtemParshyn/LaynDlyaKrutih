# Create React App. REACT_APP_* подставляется на этапе сборки.
FROM node:20-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ARG REACT_APP_API_URL=http://localhost:8080
ENV REACT_APP_API_URL=$REACT_APP_API_URL
# CRA treats warnings as errors when CI=true. Keep the production build
# from failing on lint warnings that do not break the app.
ENV CI=false
ENV DISABLE_ESLINT_PLUGIN=true

RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/build /usr/share/nginx/html

EXPOSE 3000

CMD ["nginx", "-g", "daemon off;"]
