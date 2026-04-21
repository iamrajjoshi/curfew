import { Hero } from "@/components/landing/Hero";
import { WhyCurfew } from "@/components/landing/WhyCurfew";
import { BentoFeatures } from "@/components/landing/BentoFeatures";
import { HowItWorks } from "@/components/landing/HowItWorks";
import { ConfigPreview } from "@/components/landing/ConfigPreview";
import { PrivacySection } from "@/components/landing/PrivacySection";
import { InstallCTA } from "@/components/landing/InstallCTA";
import { NavBar } from "@/components/NavBar";
import { Footer } from "@/components/Footer";

export default function Home() {
  return (
    <>
      <NavBar />
      <main>
        <Hero />
        <WhyCurfew />
        <BentoFeatures />
        <HowItWorks />
        <ConfigPreview />
        <PrivacySection />
        <InstallCTA />
      </main>
      <Footer />
    </>
  );
}
