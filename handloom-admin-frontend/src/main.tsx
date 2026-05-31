import './index.css';
// Leaflet base stylesheet — required for tile rendering on the geomap.
import 'leaflet/dist/leaflet.css';

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import App from './app/App';

const rootElement = document.getElementById('root');
if (!rootElement) throw new Error('Root element not found');

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>
);
