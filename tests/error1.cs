namespace Bad {
    class Broken {
        // TC0002: const without initializer
        public const int MaxVal;

        // TC0005: abstract with body
        public abstract void DoThing() {
            int x = 1;
        }

        // TC0009: missing return
        public int Compute(int x) {
            if (x > 0) {
                return x;
            }
            // no return on else path
        }

        // TC0037: assign to const
        public void MutateConst() {
            const int k = 5;
            k = 10;
        }

        // TC0010: break outside loop
        public void BadBreak() {
            break;
        }
    }
}