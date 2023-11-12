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
