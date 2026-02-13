import React from 'react';
import { createRoot } from 'react-dom/client';
import dayjs from 'dayjs';
import duration from 'dayjs/plugin/duration';
import 'virtual:svg-icons-register';
import './i18n';
import './style.css';
import App from './App';

dayjs.extend(duration);

const container = document.getElementById('root');

const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
