import { APP_INITIALIZER, ApplicationConfig, inject, isDevMode } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideRouter } from '@angular/router';
import { provideServiceWorker } from '@angular/service-worker';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth.interceptor';
import { FirebaseConfigService } from './core/firebase-config';
import { unauthorizedInterceptor } from './core/unauthorized.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient(withInterceptors([unauthorizedInterceptor, authInterceptor])),
    provideAnimationsAsync(),
    // Firebase settings are read at startup, before AuthService first needs them.
    {
      provide: APP_INITIALIZER,
      multi: true,
      useFactory: () => {
        const config = inject(FirebaseConfigService);
        return () => config.load();
      },
    },
    // Production builds only; `ng serve` never runs a service worker.
    provideServiceWorker('ngsw-worker.js', {
      enabled: !isDevMode(),
      registrationStrategy: 'registerWhenStable:30000',
    }),
  ],
};
