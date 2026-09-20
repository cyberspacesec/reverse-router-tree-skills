import SiteFooter from './components/SiteFooter'
import SiteHeader from './components/SiteHeader'
import Hero from './components/Hero'
import Problems from './components/Problems'
import Features from './components/Features'
import Normalization from './components/Normalization'
import QuickStart from './components/QuickStart'

export default function App() {
  return (
    <>
      <SiteHeader />
      <main>
        <Hero />
        <Problems />
        <Features />
        <Normalization />
        <QuickStart />
      </main>
      <SiteFooter />
    </>
  )
}
