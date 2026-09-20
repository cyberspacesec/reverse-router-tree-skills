import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import App from './App'
import './global.css'

// 主色对齐教学站（VitePress theme-color #3aa676），两站视觉统一。
const theme = {
  token: {
    colorPrimary: '#3aa676',
    borderRadius: 8,
  },
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider locale={zhCN} theme={theme}>
      <App />
    </ConfigProvider>
  </React.StrictMode>,
)
