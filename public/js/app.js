/**
 * @param {string} url 
 * @param {{
 *   fields: Record<string, string>
 *   validationErrors: Record<string, string> 
 * }} data
 */
async function validateForm(url, data) {
  url = `${url}?` + new URLSearchParams(data.fields)
  /** @type {{validationErrors: Record<string, string> | undefined}} */
  const response = await fetch(url).then((r) => r.json())
  data.validationErrors = response.validationErrors ?? {}
}


document.addEventListener('alpine:init', () => {
  Alpine.data('formField', (fieldName) => ({
    get valid() { return this.$data.validationErrors[fieldName] == undefined },
    get error() { return this.$data.validationErrors[fieldName] },
    get value() { return this.$data.fields[fieldName] },
    set value(newValue) { this.$data.fields[fieldName] = newValue },
  }))
})
