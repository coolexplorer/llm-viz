import type { Metadata } from 'next';
import { Orbitron, Rajdhani, Share_Tech_Mono } from 'next/font/google';
import './globals.css';

const orbitron = Orbitron({
  subsets: ['latin'],
  weight: ['400', '700', '900'],
  variable: '--font-orbitron',
  display: 'swap',
});

const rajdhani = Rajdhani({
  subsets: ['latin'],
  weight: ['300', '500', '700'],
  variable: '--font-rajdhani',
  display: 'swap',
});

const shareTech = Share_Tech_Mono({
  subsets: ['latin'],
  weight: ['400'],
  variable: '--font-share-tech',
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'llm-viz — Real-time Token Dashboard',
  description:
    'Visual-first real-time token consumption and context window dashboard for multi-provider LLM developers.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body
        className={`${orbitron.variable} ${rajdhani.variable} ${shareTech.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
