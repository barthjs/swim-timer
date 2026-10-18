import type { LocationQuery, RouteLocationRaw } from 'vue-router';

/** REDIRECT_QUERY names the query parameter that carries the page to return to after logging in. */
const REDIRECT_QUERY = 'redirect';

/** LoginRoute is the login page as a path, such as a localized one, or as a named route. */
export type LoginRoute = string | { name: string };

/** loginLocation builds the location of the login page that returns to fullPath after a successful login. */
export function loginLocation(login: LoginRoute, fullPath?: string): RouteLocationRaw {
  const page =
    typeof login === 'string'
      ? {
          path: login,
        }
      : {
          name: login.name,
        };
  if (!fullPath || fullPath === '/') {
    return page;
  }

  return {
    ...page,
    query: {
      [REDIRECT_QUERY]: fullPath,
    },
  };
}

/**
 * redirectTarget returns the page to open after a successful login, or the fallback without a usable one.
 * Only paths within the app are accepted, so a crafted login link cannot send the user to another site
 * ("//host" and "/\host" are protocol-relative URLs).
 */
export function redirectTarget(query: LocationQuery, fallback: RouteLocationRaw): RouteLocationRaw {
  const redirect = query[REDIRECT_QUERY];
  if (typeof redirect !== 'string') {
    return fallback;
  }

  if (!redirect.startsWith('/') || redirect.startsWith('//') || redirect.startsWith('/\\')) {
    return fallback;
  }

  return redirect;
}
