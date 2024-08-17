/**
 * @param {string} url
 * @param {{
 *   fields: Record<string, string>
 *   validationErrors: Record<string, string>
 * }} data
 */
async function validateForm(url, data) {
  url = `${url}?` + new URLSearchParams(data.fields);
  /** @type {{validationErrors: Record<string, string> | undefined}} */
  const response = await fetch(url).then((r) => r.json());
  data.validationErrors = response.validationErrors ?? {};
}

document.addEventListener("alpine:init", () => {
  Alpine.data("formField", (fieldName, opts = {}) => ({
    get valid() {
      return this.$data.validationErrors?.[fieldName] == undefined;
    },
    get error() {
      return this.$data.validationErrors?.[fieldName];
    },
    get value() {
      return this.$data.fields[fieldName];
    },
    set value(newValue) {
      this.$data.fields[fieldName] = newValue;
    },
    input: {
      [opts.debounce != undefined
        ? `@input.debounce.${opts.debounce}`
        : "@input.debounce"]() {
        this.$dispatch("validate");
      },
      [":id"]: "fieldId",
      ["x-model"]: "value",
      ["name"]: fieldName,
      [":aria-invalid"]: "!valid",
      [":aria-errormessage"]: "errorId",
    },
    errorId: "",
    fieldId: "",
    init() {
      this.errorId = this.$id("error");
      this.fieldId = this.$id("field");
    },
  }));

  Alpine.data("toast", ({ durationMs } = {}) => ({
    init() {
      /** @type {HTMLElement} */
      let root = this.$el;
      root.classList.remove("hidden");
      if (!durationMs) {
        durationMs = 2000;
      }
      let originalRight;
      setTimeout(() => {
        const myHeight = root.getBoundingClientRect().height;
        for (const toast of document.querySelectorAll('[component="toast"]')) {
          if (toast === root) continue;
          const toastRect = toast.getBoundingClientRect();
          toast.style.top = toastRect.top + 16 + myHeight;
        }
        originalRight = root.style.right;
        root.style.right = "1rem";
      }, 100);
      setTimeout(() => {
        root.style.right = originalRight;
      }, 100 + durationMs);
      setTimeout(
        () => {
          root.remove();
        },
        100 + durationMs + 300,
      );
    },
  }));
});
