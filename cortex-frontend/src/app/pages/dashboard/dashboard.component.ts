import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AsyncPipe, NgFor, NgIf } from '@angular/common';
import { ScanService, ScanTask, ScanMode } from '../../core/services/scan.service';
import { CardComponent } from '../../shared/card/card.component';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [FormsModule, NgIf, NgFor, AsyncPipe, CardComponent],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.scss'
})
export class DashboardComponent {
  hosts = 'scanme.nmap.org';
  ports = '22,80,443';
  mode: ScanMode = 'connect';

  tasks$ = this.scan.tasks$;

  constructor(private scan: ScanService, private router: Router) {}

  quickScan() {
    const req = { hosts: this.hosts.split(/\s|,|;/).filter(Boolean), ports: this.ports.trim(), mode: this.mode };
    this.scan.createScan(req).subscribe(({ id }) => {
      this.router.navigate(['/scans', id]);
    });
  }
}
