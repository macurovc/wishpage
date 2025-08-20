import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './App.css' // Only import App.css now

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
