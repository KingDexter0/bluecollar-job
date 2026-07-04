import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "BlueCollarJob",
  description: "Blue-collar hiring, onboarding, ATS, and interview scheduling demo"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
