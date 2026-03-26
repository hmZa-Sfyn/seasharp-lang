using System;
using System.Collections.Generic;

namespace Examples {
    class CollectionDemo {
        public void Run() {
            List<int> nums = new List<int>();
            nums.Add(1);
            nums.Add(2);
            nums.Add(3);

            foreach (int n in nums) {
                Console.WriteLine(n);
            }

            int[] arr = new int[5];
            for (int i = 0; i < arr.Length; i++) {
                arr[i] = i * i;
            }
        }
    }
}