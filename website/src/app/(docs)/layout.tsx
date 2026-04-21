import { NavBar } from "@/components/NavBar";
import { Footer } from "@/components/Footer";
import { Sidebar } from "@/components/docs/Sidebar";
import { MobileSidebar } from "@/components/docs/MobileSidebar";
import { PrevNextLinks } from "@/components/docs/PrevNextLinks";

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <NavBar />
      <div className="mx-auto flex max-w-[1280px] gap-10 px-6 pt-32 lg:px-8">
        <aside className="hidden w-64 shrink-0 xl:block">
          <div className="sticky top-24 max-h-[calc(100vh-8rem)] overflow-y-auto">
            <Sidebar />
          </div>
        </aside>

        <main className="min-w-0 max-w-[860px] flex-1 pb-20">
          <div className="mb-6 xl:hidden">
            <MobileSidebar />
          </div>

          <article className="prose max-w-none pb-16">
            {children}
          </article>
          <PrevNextLinks />
        </main>
      </div>
      <Footer />
    </>
  );
}
