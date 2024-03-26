import SwaggerUI from 'swagger-ui-dist/swagger-ui-es-bundle.js';
import 'swagger-ui-dist/swagger-ui.css';

window.addEventListener('load', async () => {
  const url = document.getElementById('swagger-ui').getAttribute('data-source');

  const ui = SwaggerUI({
    url,
    dom_id: '#swagger-ui',
    deepLinking: true,
    docExpansion: 'none',
    defaultModelRendering: 'model', // don't show examples by default, because they may be incomplete
    presets: [
      SwaggerUI.presets.apis,
    ],
    plugins: [
      SwaggerUI.plugins.DownloadUrl,
    ],
  });

  window.ui = ui;
});
