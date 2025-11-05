import { Component, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AsyncPipe, NgFor, NgIf } from '@angular/common';
import { Subscription } from 'rxjs';
import { ScanService, ScanTask } from '../../../core/services/scan.service';

@Component({
  selector: 'app-scan-result',
  standalone: true,
  imports: [NgIf, NgFor, AsyncPipe],
  templateUrl: './scan-result.component.html',
  styleUrl: './scan-result.component.scss'
})
export class ScanResultComponent implements OnDestroy {
  task?: ScanTask;
  sub?: Subscription;

  constructor(private route: ActivatedRoute, private scan: ScanService) {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.sub = this.scan.pollScan(id).subscribe((t) => (this.task = t));
  }

  ngOnDestroy() { this.sub?.unsubscribe(); }
}
