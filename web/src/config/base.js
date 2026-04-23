const normalizeBase = (value) => {
    if (!value || value === '/') return '/';

    let next = value.trim();
    if (!next.startsWith('/')) {
        next = `/${next}`;
    }
    if (!next.endsWith('/')) {
        next = `${next}/`;
    }
    return next;
};

export const appBase = normalizeBase(import.meta.env.BASE_URL);
export const appPrefix = appBase === '/' ? '' : appBase.slice(0, -1);
export const routerBasename = appPrefix || '/';

export const appPath = (path) => {
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    return `${appPrefix}${normalizedPath}` || normalizedPath;
};
