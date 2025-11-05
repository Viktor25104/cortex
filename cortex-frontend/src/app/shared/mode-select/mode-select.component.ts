import { Component, forwardRef, Input } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { NgClass } from '@angular/common';

type Mode = 'connect' | 'syn' | 'udp';

@Component({
  selector: 'app-mode-select',
  standalone: true,
  imports: [NgClass],
  templateUrl: './mode-select.component.html',
  styleUrl: './mode-select.component.scss',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => ModeSelectComponent),
      multi: true,
    },
  ],
})
export class ModeSelectComponent implements ControlValueAccessor {
  @Input() disabled = false;
  value: Mode = 'connect';
  onChange: (val: Mode) => void = () => {};
  onTouched: () => void = () => {};

  writeValue(val: Mode) {
    if (val) this.value = val;
  }
  registerOnChange(fn: any) { this.onChange = fn; }
  registerOnTouched(fn: any) { this.onTouched = fn; }
  setDisabledState(isDisabled: boolean) { this.disabled = isDisabled; }

  select(val: Mode) {
    if (this.disabled) return;
    this.value = val;
    this.onChange(val);
    this.onTouched();
  }
}

