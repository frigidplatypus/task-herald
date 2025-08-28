import flatpickr from 'flatpickr';
import type { Instance, Options } from 'flatpickr/dist/types/instance';

export function datepicker(node: HTMLInputElement, options: Options = {}): { destroy(): void } {
  const fp: Instance = flatpickr(node, options);
  // Automatically open calendar if requested
  if ((options as any).autoOpen) {
    // slight delay to ensure DOM ready
    setTimeout(() => fp.open(), 0);
  }
  return {
    destroy() {
      fp.destroy();
    }
  };
}
