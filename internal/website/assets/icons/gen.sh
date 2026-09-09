npx realfavicon generate favicon.svg favicon-settings.json output-data.json ./favicon/
touch index.html
npx realfavicon inject output-data.json . index.html

