import { NextRequest, NextResponse } from 'next/server';

const PROTECTED_PATHS = ['/checkout', '/account'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const isProtected = PROTECTED_PATHS.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`),
  );

  if (!isProtected) return NextResponse.next();

  const hasAccess = request.cookies.has('store_token');
  const hasRefresh = request.cookies.has('store_refresh');

  if (hasAccess || hasRefresh) return NextResponse.next();

  const loginUrl = new URL('/login', request.url);
  loginUrl.searchParams.set('redirect', pathname);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: ['/checkout/:path*', '/account/:path*'],
};
