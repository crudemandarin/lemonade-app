import { HttpErrorResponse, HttpInterceptorFn, HttpResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { tap } from 'rxjs';

import { RequestLogService } from './request-log.service';

/** Records every HttpClient request and its outcome in the RequestLogService. */
export const requestLogInterceptor: HttpInterceptorFn = (req, next) => {
  const log = inject(RequestLogService);
  const timestamp = new Date();
  const started = performance.now();

  const record = (status: number, responseBody: unknown) =>
    log.add({
      method: req.method,
      url: req.urlWithParams,
      status,
      durationMs: Math.round(performance.now() - started),
      timestamp,
      requestBody: req.body,
      responseBody,
    });

  return next(req).pipe(
    tap({
      next: (event) => {
        if (event instanceof HttpResponse) {
          record(event.status, event.body);
        }
      },
      error: (err: unknown) => {
        if (err instanceof HttpErrorResponse) {
          record(err.status, err.error);
        }
      },
    }),
  );
};
