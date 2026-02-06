import React from 'react';
import { createRoot } from 'react-dom/client';
import 'virtual:svg-icons-register';
import './i18n';
import './style.css';
import App from './App';

const container = document.getElementById('root');

const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
