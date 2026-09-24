import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

import { Sample } from '../models/sample.model';

const API_URL = '/api';

/** Thin HTTP client for the lemonade-api API. */
@Injectable({ providedIn: 'root' })
export class SampleService {
  private readonly http = inject(HttpClient);

  health(): Observable<string> {
    return this.http.get(`${API_URL}/`, { responseType: 'text' });
  }

  list(): Observable<Sample[]> {
    return this.http.get<Sample[]>(`${API_URL}/samples`);
  }

  get(id: number): Observable<Sample> {
    return this.http.get<Sample>(`${API_URL}/samples/${id}`);
  }

  create(name: string): Observable<Sample> {
    return this.http.post<Sample>(`${API_URL}/samples`, { name });
  }

  update(id: number, name: string): Observable<Sample> {
    return this.http.put<Sample>(`${API_URL}/samples/${id}`, { name });
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${API_URL}/samples/${id}`);
  }
}
