/** A text box's value in a submitted form; empty if the form has no such text box. */
export function textField(form: FormData, name: string): string {
  const value = form.get(name);
  return typeof value === 'string' ? value : '';
}
