import Link from "next/link";

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center px-6 text-center">
      <h1 className="font-heading text-4xl font-medium tracking-tight text-curfew-text sm:text-5xl">
        404
      </h1>
      <p className="mt-3 max-w-md text-base text-curfew-muted">
        This page wandered past bedtime.
      </p>
      <Link
        href="/"
        className="mt-6 inline-flex rounded-full border border-brand-purple/25 bg-brand-purple/10 px-4 py-2 text-sm font-medium text-brand-purple transition-colors hover:bg-brand-purple/14"
      >
        Back to curfew
      </Link>
    </main>
  );
}
