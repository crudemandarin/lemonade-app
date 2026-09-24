import { Component } from '@angular/core';

import { ApiStatusComponent } from './components/api-status/api-status.component';
import { CreateSampleComponent } from './components/create-sample/create-sample.component';
import { RequestHistoryComponent } from './components/request-history/request-history.component';
import { SampleLookupComponent } from './components/sample-lookup/sample-lookup.component';
import { SampleTableComponent } from './components/sample-table/sample-table.component';

@Component({
  selector: 'app-index',
  standalone: true,
  imports: [
    ApiStatusComponent,
    CreateSampleComponent,
    RequestHistoryComponent,
    SampleLookupComponent,
    SampleTableComponent,
  ],
  templateUrl: './index.component.html',
  styleUrl: './index.component.scss',
})
export class IndexComponent {}
