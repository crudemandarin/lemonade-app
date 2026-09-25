import { TestBed } from '@angular/core/testing';

import { FIREBASE_CONFIG_URL, FirebaseConfigService } from './firebase-config';

describe('FirebaseConfigService', () => {
  const good = { apiKey: 'k', authDomain: 'a.test', projectId: 'p', appId: 'app' };

  let fetchSpy: jasmine.Spy;

  beforeEach(() => (fetchSpy = spyOn(window, 'fetch')));

  function stubFetch(response: () => Promise<Response>) {
    return fetchSpy.and.callFake(response);
  }

  it('loads the config at startup without using the HTTP cache', async () => {
    const fetch = stubFetch(async () => new Response(JSON.stringify(good)));
    const service = TestBed.inject(FirebaseConfigService);

    await service.load();

    expect(service.config).toEqual(good);
    expect(fetch).toHaveBeenCalledWith(FIREBASE_CONFIG_URL, { cache: 'no-store' });
  });

  it('is served from outside /assets, which the service worker caches', () => {
    expect(FIREBASE_CONFIG_URL.startsWith('/assets/')).toBeFalse();
  });

  it('leaves sign-in unconfigured when the file is missing, invalid or unreachable', async () => {
    const service = TestBed.inject(FirebaseConfigService);
    for (const response of [
      async () => new Response('nope', { status: 404 }),
      async () => new Response(JSON.stringify({ ...good, apiKey: '' })),
      async () => new Response('<html>'),
      async () => {
        throw new Error('offline');
      },
    ]) {
      stubFetch(response);
      await service.load();
      expect(service.config).toBeNull();
    }
  });
});
