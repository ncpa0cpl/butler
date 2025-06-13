class TabBtn {
  /** @type {TabsElement} */
  parent;
  /** @type {HTMLButtonElement} */
  btn;
  /** @type {string} */
  tabID;

  /**
   * @param {TabsElement} parent
   * @param {HTMLButtonElement} btn
   * @param {string} tabID
   */
  constructor(parent, btn, tabID) {
    this.parent = parent;
    this.btn = btn;
    this.tabID = tabID;
    btn.addEventListener("click", () => {
      parent.openTab(tabID);
    });
  }

  /**
   * @param {bool} active
   */
  setActive(active) {
    const activeClasses = ["bg-white", "text-black", "shadow-sm", "dark:bg-slate-800", "dark:text-gray-100"];
    if (active) {
      this.btn.classList.add(...activeClasses);
    } else {
      this.btn.classList.remove(...activeClasses);
    }
  }
}

class TabContent {
  /** @type {TabsElement} */
  parent;
  /** @type {HTMLElement} */
  elem;
  /** @type {string} */
  tabID;

  /**
   * @param {TabsElement} parent
   * @param {HTMLButtonElement} btn
   * @param {string} tabID
   */
  constructor(parent, elem, tabID) {
    this.parent = parent;
    this.elem = elem;
    this.tabID = tabID;
  }

  hide() {
    this.elem.classList.add("hidden");
  }

  show() {
    this.elem.classList.remove("hidden");
    this.parent.setActiveTabID(this.tabID);
  }
}

class TabsElement extends HTMLDivElement {
  /** @type {TabBtn[]} */
  buttons = [];
  /** @type {Array<TabContent>} */
  contentElems = [];
  activeTab = "";

  /** @param {string} id */
  setActiveTabID(id) {
    this.activeTab = id;
    for (const btn of this.buttons) {
      if (btn.tabID === id) {
        btn.setActive(true);
      } else {
        btn.setActive(false);
      }
    }
  }

  /** @param {string} tabID  */
  openTab(tabID) {
    for (const tab of this.contentElems) {
      if (tab.tabID === tabID) {
        tab.show();
      } else {
        tab.hide();
      }
    }
  }

  connectedCallback() {
    /** @type {NodeListOf<HTMLButtonElement>} */
    const buttons = this.querySelectorAll(".tab-buttons > button");
    for (const btn of buttons) {
      const tabID = btn.dataset.tabid;
      if (tabID) {
        this.buttons.push(new TabBtn(this, btn, tabID));
      }
    }

    const contents = this.querySelectorAll(".tab-content");
    for (const content of contents) {
      const tabID = content.dataset.tabid;
      const btn = this.buttons.find((btn) => btn.tabID === tabID);
      if (btn && tabID) {
        const tab = new TabContent(this, content, tabID);
        this.contentElems.push(tab);
        if (this.activeTab === "") {
          tab.show();
        }
      }
    }
  }
}

customElements.define("cst-tabs", TabsElement, { extends: "div" });

/** @type {GroupBtn[]} */
const allGroupBtns = [];

class GroupBtn extends HTMLButtonElement {
  setActive(active) {
    const activeClasses = ["bg-gray-100", "dark:bg-slate-700", "text-primary", "text-black", "dark:text-white"];
    const inactiveClasses = ["text-gray-600", "dark:text-gray-300"];

    if (active) {
      this.classList.remove(...inactiveClasses);
      this.classList.add(...activeClasses);
    } else {
      this.classList.remove(...activeClasses);
      this.classList.add(...inactiveClasses);
    }
  }

  connectedCallback() {
    const endpId = this.dataset.endpid;
    if (!endpId) return;

    /** @type {Element} */
    let ownDetails;

    const endpointDetails = document.querySelectorAll(".endp-details");
    for (const endpoint of endpointDetails) {
      const id = endpoint.dataset.endpid;
      if (id === endpId) {
        ownDetails = endpoint;
      }
    }

    this.addEventListener("click", () => {
      for (const endpoint of endpointDetails) {
        endpoint.classList.add("hidden");
      }
      ownDetails.classList.remove("hidden");

      this.setActive(true);
      for (const btn of allGroupBtns) {
        if (btn != this) {
          btn.setActive(false);
        }
      }
    });

    if (!ownDetails.classList.contains("hidden")) {
      this.setActive(true);
    }

    allGroupBtns.push(this);
  }
}

customElements.define("cst-groupbtn", GroupBtn, { extends: "button" });

class CollapsibleBtn extends HTMLButtonElement {
  connectedCallback() {
    const id = this.dataset.id;
    if (id == null) {
      return;
    }

    const chevron = this.querySelector("svg")
    chevron.style.transition = "transform .1s"

    const allcontainers = document.querySelectorAll(".collapsible-content");
    const container = Array.from(allcontainers).find((c) => {
      const containerID = c.dataset.id;
      return containerID == id;
    });

    if (container == null) {
      return;
    }

    let isopen = !container.classList.contains("hidden");
    this.addEventListener("click", () => {
      if (isopen) {
        container.classList.add("hidden");
        chevron.style.transform =""
      } else {
        container.classList.remove("hidden");
        chevron.style.transform ="rotate(90deg)"
      }
      isopen = !isopen;
    });
  }
}

customElements.define("cst-collapsiblebtn", CollapsibleBtn, { extends: "button" });
