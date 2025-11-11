import { HttpInterceptorFn } from '@angular/common/http';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const apiKey = localStorage.getItem('cortex_api_key');
  if (apiKey) {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${apiKey}` } });
  }
  return next(req);
};
