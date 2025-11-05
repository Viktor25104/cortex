import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AsyncPipe, NgFor } from '@angular/common';
import { ScanService, ScanMode } from '../../../core/services/scan.service';

@Component({
  selector: 'app-scan-manager',
  standalone: true,
  imports: [FormsModule, NgFor, AsyncPipe, RouterLink],
  templateUrl: './scan-manager.component.html',
  styleUrl: './scan-manager.component.scss'
})
export class ScanManagerComponent {
  hosts = '';
  ports = '22,80,443';
  mode: ScanMode = 'connect';
  tasks$ = this.scan.tasks$;

  constructor(private scan: ScanService, private router: Router) {}

  launch() {
    const req = { hosts: this.hosts.split(/\s|,|;/).filter(Boolean), ports: this.ports.trim(), mode: this.mode };
    this.scan.createScan(req).subscribe(({ id }) => this.router.navigate(['/scans', id]));
  }
}
